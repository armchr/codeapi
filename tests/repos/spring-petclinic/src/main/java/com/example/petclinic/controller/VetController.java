package com.example.petclinic.controller;

import com.example.petclinic.dto.VetDto;
import com.example.petclinic.service.VetService;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.util.List;

/**
 * REST controller for Vet operations.
 */
@RestController
@RequestMapping("/api/vets")
public class VetController {

    private final VetService vetService;

    public VetController(VetService vetService) {
        this.vetService = vetService;
    }

    /**
     * GET /api/vets - Get all vets.
     */
    @GetMapping
    public ResponseEntity<List<VetDto>> getAllVets() {
        return ResponseEntity.ok(vetService.findAll());
    }

    /**
     * GET /api/vets/{id} - Get vet by ID.
     */
    @GetMapping("/{id}")
    public ResponseEntity<VetDto> getVetById(@PathVariable Long id) {
        return ResponseEntity.ok(vetService.findById(id));
    }

    /**
     * GET /api/vets/specialty/{specialtyName} - Get vets by specialty.
     */
    @GetMapping("/specialty/{specialtyName}")
    public ResponseEntity<List<VetDto>> getVetsBySpecialty(@PathVariable String specialtyName) {
        return ResponseEntity.ok(vetService.findBySpecialty(specialtyName));
    }

    /**
     * POST /api/vets - Create a new vet.
     */
    @PostMapping
    public ResponseEntity<VetDto> createVet(@Valid @RequestBody VetDto vetDto) {
        VetDto created = vetService.create(vetDto);
        return new ResponseEntity<>(created, HttpStatus.CREATED);
    }

    /**
     * POST /api/vets/{id}/specialties - Add specialty to a vet.
     */
    @PostMapping("/{id}/specialties")
    public ResponseEntity<VetDto> addSpecialty(@PathVariable Long id,
                                                @RequestParam String specialtyName) {
        return ResponseEntity.ok(vetService.addSpecialty(id, specialtyName));
    }

    /**
     * DELETE /api/vets/{id} - Delete a vet.
     */
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteVet(@PathVariable Long id) {
        vetService.delete(id);
        return ResponseEntity.noContent().build();
    }
}
