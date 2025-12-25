package com.example.petclinic.service;

import com.example.petclinic.dto.VetDto;
import com.example.petclinic.exception.ResourceNotFoundException;
import com.example.petclinic.model.Specialty;
import com.example.petclinic.model.Vet;
import com.example.petclinic.repository.SpecialtyRepository;
import com.example.petclinic.repository.VetRepository;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

/**
 * Service layer for Vet operations.
 */
@Service
@Transactional
public class VetService {

    private final VetRepository vetRepository;
    private final SpecialtyRepository specialtyRepository;

    public VetService(VetRepository vetRepository, SpecialtyRepository specialtyRepository) {
        this.vetRepository = vetRepository;
        this.specialtyRepository = specialtyRepository;
    }

    /**
     * Find all vets with their specialties.
     */
    @Transactional(readOnly = true)
    @Cacheable("vets")
    public List<VetDto> findAll() {
        return vetRepository.findAllWithSpecialties().stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find vet by ID.
     */
    @Transactional(readOnly = true)
    public VetDto findById(Long id) {
        Vet vet = vetRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Vet", "id", id));
        return toDto(vet);
    }

    /**
     * Find vets by specialty.
     */
    @Transactional(readOnly = true)
    public List<VetDto> findBySpecialty(String specialtyName) {
        return vetRepository.findBySpecialtyName(specialtyName).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Create a new vet.
     */
    public VetDto create(VetDto vetDto) {
        Vet vet = new Vet();
        vet.setFirstName(vetDto.firstName());
        vet.setLastName(vetDto.lastName());

        if (vetDto.specialties() != null) {
            for (String specialtyName : vetDto.specialties()) {
                Specialty specialty = specialtyRepository.findByName(specialtyName)
                    .orElseGet(() -> {
                        Specialty newSpecialty = new Specialty(specialtyName);
                        return specialtyRepository.save(newSpecialty);
                    });
                vet.addSpecialty(specialty);
            }
        }

        Vet savedVet = vetRepository.save(vet);
        return toDto(savedVet);
    }

    /**
     * Add specialty to a vet.
     */
    public VetDto addSpecialty(Long vetId, String specialtyName) {
        Vet vet = vetRepository.findById(vetId)
            .orElseThrow(() -> new ResourceNotFoundException("Vet", "id", vetId));

        Specialty specialty = specialtyRepository.findByName(specialtyName)
            .orElseGet(() -> {
                Specialty newSpecialty = new Specialty(specialtyName);
                return specialtyRepository.save(newSpecialty);
            });

        vet.addSpecialty(specialty);
        Vet savedVet = vetRepository.save(vet);
        return toDto(savedVet);
    }

    /**
     * Delete a vet.
     */
    public void delete(Long id) {
        if (!vetRepository.existsById(id)) {
            throw new ResourceNotFoundException("Vet", "id", id);
        }
        vetRepository.deleteById(id);
    }

    private VetDto toDto(Vet vet) {
        Set<String> specialtyNames = vet.getSpecialties().stream()
            .map(Specialty::getName)
            .collect(Collectors.toSet());

        return new VetDto(
            vet.getId(),
            vet.getFirstName(),
            vet.getLastName(),
            specialtyNames
        );
    }
}
