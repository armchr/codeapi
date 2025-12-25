package com.example.petclinic.repository;

import com.example.petclinic.model.Pet;
import com.example.petclinic.model.PetType;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.time.LocalDate;
import java.util.List;

/**
 * Repository for Pet entity.
 */
@Repository
public interface PetRepository extends JpaRepository<Pet, Long> {

    /**
     * Find pets by owner ID.
     */
    List<Pet> findByOwnerId(Long ownerId);

    /**
     * Find pets by type.
     */
    List<Pet> findByType(PetType type);

    /**
     * Find pets by name containing.
     */
    List<Pet> findByNameContainingIgnoreCase(String name);

    /**
     * Find pets born after a certain date.
     */
    List<Pet> findByBirthDateAfter(LocalDate date);

    /**
     * Find pets with visits.
     */
    @Query("SELECT DISTINCT p FROM Pet p LEFT JOIN FETCH p.visits WHERE p.owner.id = :ownerId")
    List<Pet> findByOwnerIdWithVisits(@Param("ownerId") Long ownerId);

    /**
     * Count pets by type.
     */
    @Query("SELECT COUNT(p) FROM Pet p WHERE p.type.name = :typeName")
    long countByTypeName(@Param("typeName") String typeName);
}
