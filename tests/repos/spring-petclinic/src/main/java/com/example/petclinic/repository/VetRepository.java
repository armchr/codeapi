package com.example.petclinic.repository;

import com.example.petclinic.model.Vet;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.util.List;

/**
 * Repository for Vet entity.
 */
@Repository
public interface VetRepository extends JpaRepository<Vet, Long> {

    /**
     * Find vets by last name.
     */
    List<Vet> findByLastNameContainingIgnoreCase(String lastName);

    /**
     * Find vets with a specific specialty.
     */
    @Query("SELECT v FROM Vet v JOIN v.specialties s WHERE s.name = :specialtyName")
    List<Vet> findBySpecialtyName(@Param("specialtyName") String specialtyName);

    /**
     * Find all vets with their specialties eagerly loaded.
     */
    @Query("SELECT DISTINCT v FROM Vet v LEFT JOIN FETCH v.specialties")
    List<Vet> findAllWithSpecialties();

    /**
     * Count vets by specialty.
     */
    @Query("SELECT COUNT(v) FROM Vet v JOIN v.specialties s WHERE s.name = :specialtyName")
    long countBySpecialtyName(@Param("specialtyName") String specialtyName);
}
